package embedding

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	Dimensions  = 384
	MaxChunkLen = 512
	ORTVersion  = "1.23.2"

	modelURL    = "https://huggingface.co/onnx-models/all-MiniLM-L6-v2-onnx/resolve/main/model.onnx"
	tokenizerURL = "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json"
)

var (
	mu        sync.Mutex
	modelDir  string
	embedder  *Embedder
)

type Embedder struct {
	session *ort.DynamicAdvancedSession
	tok     *tokenizer.Tokenizer
}

func (e *Embedder) Compute(text string) ([]float32, error) {
	results, err := e.ComputeBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results")
	}
	return results[0], nil
}

func (e *Embedder) ComputeBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	inputBatch := make([]tokenizer.EncodeInput, len(texts))
	for i, s := range texts {
		inputBatch[i] = tokenizer.NewSingleEncodeInput(tokenizer.NewRawInputSequence(s))
	}

	encodings, err := e.tok.EncodeBatch(inputBatch, false)
	if err != nil {
		return nil, fmt.Errorf("tokenize: %w", err)
	}

	batchSize := len(encodings)
	seqLength := len(encodings[0].Ids)

	inputIdsData := make([]int64, batchSize*seqLength)
	attentionMaskData := make([]int64, batchSize*seqLength)
	tokenTypeIdsData := make([]int64, batchSize*seqLength)

	for b := range batchSize {
		for i, id := range encodings[b].Ids {
			inputIdsData[b*seqLength+i] = int64(id)
		}
		for i, mask := range encodings[b].AttentionMask {
			attentionMaskData[b*seqLength+i] = int64(mask)
		}
		for i, typeId := range encodings[b].TypeIds {
			tokenTypeIdsData[b*seqLength+i] = int64(typeId)
		}
	}

	inputShape := ort.NewShape(int64(batchSize), int64(seqLength))

	inputIdsTensor, err := ort.NewTensor(inputShape, inputIdsData)
	if err != nil {
		return nil, fmt.Errorf("input_ids tensor: %w", err)
	}
	defer inputIdsTensor.Destroy()

	attentionMaskTensor, err := ort.NewTensor(inputShape, attentionMaskData)
	if err != nil {
		return nil, fmt.Errorf("attention_mask tensor: %w", err)
	}
	defer attentionMaskTensor.Destroy()

	tokenTypeIdsTensor, err := ort.NewTensor(inputShape, tokenTypeIdsData)
	if err != nil {
		return nil, fmt.Errorf("token_type_ids tensor: %w", err)
	}
	defer tokenTypeIdsTensor.Destroy()

	outputs := []ort.Value{nil, nil}

	err = e.session.Run(
		[]ort.Value{inputIdsTensor, attentionMaskTensor, tokenTypeIdsTensor},
		outputs,
	)
	if err != nil {
		return nil, fmt.Errorf("run session: %w", err)
	}
	defer outputs[0].Destroy()
	defer outputs[1].Destroy()

	sentenceEmbTensor, ok := outputs[1].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type for sentence_embedding")
	}
	flat := sentenceEmbTensor.GetData()
	results := make([][]float32, batchSize)
	for i := range batchSize {
		start := i * Dimensions
		end := start + Dimensions
		results[i] = make([]float32, Dimensions)
		copy(results[i], flat[start:end])
	}
	return results, nil
}

func (e *Embedder) Close() {
	if e.session != nil {
		e.session.Destroy()
	}
}

func InitEmbedder() (*Embedder, error) {
	mu.Lock()
	defer mu.Unlock()

	if embedder != nil {
		return embedder, nil
	}

	libPath, err := ensureORTLib()
	if err != nil {
		return nil, fmt.Errorf("ensure onnxruntime: %w", err)
	}
	ort.SetSharedLibraryPath(libPath)

	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("init ort: %w", err)
	}

	modelPath, err := ensureFile("model.onnx", modelURL)
	if err != nil {
		return nil, fmt.Errorf("ensure model: %w", err)
	}

	tokenizerPath, err := ensureFile("tokenizer.json", tokenizerURL)
	if err != nil {
		return nil, fmt.Errorf("ensure tokenizer: %w", err)
	}

	tk, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	inputNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	outputNames := []string{"token_embeddings", "sentence_embedding"}

	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, nil)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	embedder = &Embedder{session: session, tok: tk}
	return embedder, nil
}

func CloseEmbedder() {
	mu.Lock()
	defer mu.Unlock()
	if embedder != nil {
		embedder.Close()
		embedder = nil
	}
	ort.DestroyEnvironment()
}

func ensureFile(name, url string) (string, error) {
	dir := filesDir()
	dst := filepath.Join(dir, name)
	if info, err := os.Stat(dst); err == nil && info.Size() > 0 {
		return dst, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "downloading %s\n", name)
	return dst, downloadFile(dst, url)
}

func downloadFile(dst, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func filesDir() string {
	if modelDir != "" {
		return modelDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai-memory")
}

func ensureORTLib() (string, error) {
	dir := libDir()
	name := ortLibName()
	dst := filepath.Join(dir, name)

	if info, err := os.Stat(dst); err == nil && info.Size() > 0 {
		return dst, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create lib dir: %w", err)
	}

	zipURL := ortDownloadURL()
	fmt.Fprintf(os.Stderr, "downloading onnxruntime %s\n", ORTVersion)

	resp, err := http.Get(zipURL)
	if err != nil {
		return "", fmt.Errorf("download onnxruntime: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d downloading onnxruntime", resp.StatusCode)
	}

	tmpZip := dst + ".zip"
	f, err := os.Create(tmpZip)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpZip)
		return "", err
	}
	f.Close()

	if err := extractORTLib(tmpZip, dst); err != nil {
		os.Remove(tmpZip)
		return "", err
	}
	os.Remove(tmpZip)
	return dst, nil
}

func extractORTLib(zipPath, dst string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	libName := ortLibName()
	for _, file := range zr.File {
		base := filepath.Base(file.Name)
		if strings.EqualFold(base, libName) {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.Create(dst)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("%s not found in archive", libName)
}

func ortDownloadURL() string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-win-x64-%s.zip", ORTVersion, ORTVersion)
	case "linux":
		return fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-linux-x64-%s.tgz", ORTVersion, ORTVersion)
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-arm64-%s.tgz", ORTVersion, ORTVersion)
		}
		return fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-x86_64-%s.tgz", ORTVersion, ORTVersion)
	default:
		return fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-linux-x64-%s.tgz", ORTVersion, ORTVersion)
	}
}

func ortLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "onnxruntime.dll"
	case "darwin":
		return "libonnxruntime.dylib"
	default:
		return "libonnxruntime.so"
	}
}

func libDir() string {
	if modelDir != "" {
		return filepath.Join(modelDir, "lib")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai-memory", "lib")
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(na) * math.Sqrt(nb)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

func Float32ToBytes(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		bits := math.Float32bits(v)
		buf[i*4] = byte(bits)
		buf[i*4+1] = byte(bits >> 8)
		buf[i*4+2] = byte(bits >> 16)
		buf[i*4+3] = byte(bits >> 24)
	}
	return buf
}

func BytesToFloat32(buf []byte) []float32 {
	vec := make([]float32, len(buf)/4)
	for i := range vec {
		bits := uint32(buf[i*4]) | uint32(buf[i*4+1])<<8 | uint32(buf[i*4+2])<<16 | uint32(buf[i*4+3])<<24
		vec[i] = math.Float32frombits(bits)
	}
	return vec
}

type Chunk struct {
	Heading string
	Content string
}

func ChunkText(heading, content string) []Chunk {
	lines := splitLines(content)
	var chunks []Chunk
	current := heading
	var buf []string

	for _, line := range lines {
		if h := detectHeading(line); h != "" {
			if len(buf) > 0 {
				text := joinLines(buf)
				if len(text) > MaxChunkLen {
					text = text[:MaxChunkLen]
				}
				chunks = append(chunks, Chunk{Heading: current, Content: text})
				buf = nil
			}
			current = h
		}
		buf = append(buf, line)
	}
	if len(buf) > 0 {
		text := joinLines(buf)
		if len(text) > MaxChunkLen {
			text = text[:MaxChunkLen]
		}
		chunks = append(chunks, Chunk{Heading: current, Content: text})
	}
	return chunks
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	var buf bytes.Buffer
	for _, l := range lines {
		buf.WriteString(l)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func detectHeading(line string) string {
	if len(line) > 2 && line[0] == '#' && line[1] == ' ' {
		return line[2:]
	}
	if len(line) > 3 && line[0] == '#' && line[1] == '#' && line[2] == ' ' {
		return line[3:]
	}
	return ""
}

func init() {
	if runtime.GOOS == "windows" {
		modelDir = filepath.Join(os.Getenv("APPDATA"), "ai-memory")
	} else {
		home, _ := os.UserHomeDir()
		modelDir = filepath.Join(home, ".ai-memory")
	}
}
