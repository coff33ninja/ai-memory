package rag

import (
	"fmt"
	"os"

	"github.com/coff33ninja/ai-memory/internal/db"
	"github.com/coff33ninja/ai-memory/internal/embedding"
)

type Searcher struct {
	db       *db.DB
	sharedDB *db.DB
	emb      *embedding.Embedder
}

func New(d *db.DB, emb *embedding.Embedder) *Searcher {
	return &Searcher{db: d, emb: emb}
}

func (s *Searcher) SetSharedDB(shared *db.DB) {
	s.sharedDB = shared
}

func (s *Searcher) SearchMemories(queryEmb []float32, topK int) ([]db.SearchResult, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, date, experience, lesson, impact, embedding FROM memories WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return nil, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	type scored struct {
		id         int64
		date       string
		experience string
		lesson     string
		impact     string
		score      float64
	}

	var results []scored
	for rows.Next() {
		var id int64
		var date, experience, lesson, impact string
		var embBlob []byte
		if err := rows.Scan(&id, &date, &experience, &lesson, &impact, &embBlob); err != nil {
			continue
		}
		emb := embedding.BytesToFloat32(embBlob)
		score := embedding.CosineSimilarity(queryEmb, emb)
		results = append(results, scored{id, date, experience, lesson, impact, score})
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	if topK > len(results) {
		topK = len(results)
	}

	out := make([]db.SearchResult, topK)
	for i, r := range results[:topK] {
		out[i] = db.SearchResult{
			Type:    "memory",
			ID:      r.id,
			Title:   fmt.Sprintf("%s — %s", r.date, r.experience),
			Content: r.lesson,
			Score:   r.score,
		}
	}
	return out, nil
}

func (s *Searcher) SearchSkills(queryEmb []float32, topK int) ([]db.SearchResult, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, name, description, embedding FROM skills WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return nil, fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()

	type scoredSkill struct {
		id          int64
		name        string
		description string
		score       float64
	}

	var results []scoredSkill
	for rows.Next() {
		var id int64
		var name, desc string
		var embBlob []byte
		if err := rows.Scan(&id, &name, &desc, &embBlob); err != nil {
			continue
		}
		emb := embedding.BytesToFloat32(embBlob)
		score := embedding.CosineSimilarity(queryEmb, emb)
		results = append(results, scoredSkill{id, name, desc, score})
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	if topK > len(results) {
		topK = len(results)
	}

	out := make([]db.SearchResult, topK)
	for i, r := range results[:topK] {
		out[i] = db.SearchResult{
			Type:    "skill",
			ID:      r.id,
			Title:   r.name,
			Content: r.description,
			Score:   r.score,
		}
	}
	return out, nil
}

func (s *Searcher) SearchAll(queryEmb []float32, topK int) ([]db.SearchResult, error) {
	memResults, err := s.SearchMemories(queryEmb, topK)
	if err != nil {
		return nil, err
	}
	skillResults, err := s.SearchSkills(queryEmb, topK)
	if err != nil {
		return nil, err
	}

	// Search shared memories if available
	if s.sharedDB != nil {
		sharedResults, err := s.searchSharedMemories(queryEmb, topK)
		if err == nil {
			memResults = append(memResults, sharedResults...)
		}
	}

	combined := append(memResults, skillResults...)
	sortResults(combined)
	if topK > len(combined) {
		topK = len(combined)
	}
	return combined[:topK], nil
}

func (s *Searcher) IndexMemories() (int, error) {
	if s.emb == nil {
		return 0, fmt.Errorf("embedder not initialized")
	}

	conn := s.db.Conn()
	rows, err := conn.Query("SELECT id, experience, lesson FROM memories WHERE embedding IS NULL")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var experience, lesson string
		if err := rows.Scan(&id, &experience, &lesson); err != nil {
			continue
		}
		text := experience + " " + lesson
		if len(text) > embedding.MaxChunkLen {
			text = text[:embedding.MaxChunkLen]
		}
		emb, err := s.emb.Compute(text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: embed memory %d: %v\n", id, err)
			continue
		}
		blob := embedding.Float32ToBytes(emb)
		conn.Exec("UPDATE memories SET embedding = ? WHERE id = ?", blob, id)
		count++
	}
	return count, nil
}

func (s *Searcher) IndexSkills() (int, error) {
	if s.emb == nil {
		return 0, fmt.Errorf("embedder not initialized")
	}

	conn := s.db.Conn()
	rows, err := conn.Query("SELECT id, name, description, body FROM skills WHERE embedding IS NULL")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var name, desc, body string
		if err := rows.Scan(&id, &name, &desc, &body); err != nil {
			continue
		}
		text := name + " " + desc + " " + body
		if len(text) > embedding.MaxChunkLen {
			text = text[:embedding.MaxChunkLen]
		}
		emb, err := s.emb.Compute(text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: embed skill %d: %v\n", id, err)
			continue
		}
		blob := embedding.Float32ToBytes(emb)
		conn.Exec("UPDATE skills SET embedding = ? WHERE id = ?", blob, id)
		count++
	}
	return count, nil
}

func (s *Searcher) Reindex() (int, int, error) {
	conn := s.db.Conn()
	conn.Exec("UPDATE memories SET embedding = NULL")
	conn.Exec("UPDATE skills SET embedding = NULL")
	memCount, _ := s.IndexMemories()
	skillCount, _ := s.IndexSkills()

	// Also reindex shared if available
	if s.sharedDB != nil {
		s.sharedDB.Conn().Exec("UPDATE memories SET embedding = NULL")
		s.IndexSharedMemories()
	}

	return memCount, skillCount, nil
}

// EmbedMemory generates an embedding for a single memory by ID and updates it in the persona DB.
func (s *Searcher) EmbedMemory(id int64, text string) {
	if s.emb == nil {
		return
	}
	if len(text) > embedding.MaxChunkLen {
		text = text[:embedding.MaxChunkLen]
	}
	emb, err := s.emb.Compute(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: embed memory %d: %v\n", id, err)
		return
	}
	blob := embedding.Float32ToBytes(emb)
	s.db.Conn().Exec("UPDATE memories SET embedding = ? WHERE id = ?", blob, id)
}

// EmbedSharedMemory generates an embedding for a single memory by ID in the shared DB.
func (s *Searcher) EmbedSharedMemory(id int64, text string) {
	if s.emb == nil || s.sharedDB == nil {
		return
	}
	if len(text) > embedding.MaxChunkLen {
		text = text[:embedding.MaxChunkLen]
	}
	emb, err := s.emb.Compute(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: embed shared memory %d: %v\n", id, err)
		return
	}
	blob := embedding.Float32ToBytes(emb)
	s.sharedDB.Conn().Exec("UPDATE memories SET embedding = ? WHERE id = ?", blob, id)
}

func (s *Searcher) searchSharedMemories(queryEmb []float32, topK int) ([]db.SearchResult, error) {
	if s.sharedDB == nil {
		return nil, nil
	}
	rows, err := s.sharedDB.Conn().Query(
		"SELECT id, date, experience, lesson, impact, embedding FROM memories WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		id         int64
		date       string
		experience string
		lesson     string
		impact     string
		score      float64
	}

	var results []scored
	for rows.Next() {
		var id int64
		var date, experience, lesson, impact string
		var embBlob []byte
		if err := rows.Scan(&id, &date, &experience, &lesson, &impact, &embBlob); err != nil {
			continue
		}
		emb := embedding.BytesToFloat32(embBlob)
		score := embedding.CosineSimilarity(queryEmb, emb)
		results = append(results, scored{id, date, experience, lesson, impact, score})
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	if topK > len(results) {
		topK = len(results)
	}

	out := make([]db.SearchResult, topK)
	for i, r := range results[:topK] {
		out[i] = db.SearchResult{
			Type:    "shared",
			ID:      r.id,
			Title:   fmt.Sprintf("[shared] %s — %s", r.date, r.experience),
			Content: r.lesson,
			Score:   r.score,
		}
	}
	return out, nil
}

func (s *Searcher) IndexSharedMemories() (int, error) {
	if s.sharedDB == nil || s.emb == nil {
		return 0, nil
	}
	conn := s.sharedDB.Conn()
	rows, err := conn.Query("SELECT id, experience, lesson FROM memories WHERE embedding IS NULL")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var experience, lesson string
		if err := rows.Scan(&id, &experience, &lesson); err != nil {
			continue
		}
		text := experience + " " + lesson
		if len(text) > embedding.MaxChunkLen {
			text = text[:embedding.MaxChunkLen]
		}
		emb, err := s.emb.Compute(text)
		if err != nil {
			continue
		}
		blob := embedding.Float32ToBytes(emb)
		conn.Exec("UPDATE memories SET embedding = ? WHERE id = ?", blob, id)
		count++
	}
	return count, nil
}

func sortResults(items []db.SearchResult) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Score > items[j-1].Score; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
