package graph

import (
	"testing"

	"github.com/coff33ninja/ai-memory/internal/db"
)

func setupTestDB(t *testing.T) *Graph {
	t.Helper()
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return New(d.Conn())
}

func TestAddAndGetNode(t *testing.T) {
	g := setupTestDB(t)

	id, err := g.AddNode("memory", "test memory", "private", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	node, err := g.GetNode(id)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if node == nil {
		t.Fatal("node not found")
	}
	if node.Type != "memory" {
		t.Errorf("type = %q, want %q", node.Type, "memory")
	}
	if node.Label != "test memory" {
		t.Errorf("label = %q, want %q", node.Label, "test memory")
	}
	if node.Properties["key"] != "value" {
		t.Errorf("properties[key] = %q, want %q", node.Properties["key"], "value")
	}
}

func TestGetNodeByLabel(t *testing.T) {
	g := setupTestDB(t)

	g.AddNode("tool", "click", "private", nil)

	node, err := g.GetNodeByLabel("tool", "click", "private")
	if err != nil {
		t.Fatalf("GetNodeByLabel: %v", err)
	}
	if node == nil {
		t.Fatal("node not found")
	}
	if node.Label != "click" {
		t.Errorf("label = %q, want %q", node.Label, "click")
	}
}

func TestAddAndGetEdge(t *testing.T) {
	g := setupTestDB(t)

	id1, _ := g.AddNode("memory", "mem1", "private", nil)
	id2, _ := g.AddNode("memory", "mem2", "private", nil)

	edgeID, err := g.AddEdge(id1, id2, "thematic", nil, 0.8)
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if edgeID <= 0 {
		t.Fatalf("expected positive edge id, got %d", edgeID)
	}

	edges, err := g.GetEdges(id1, "outgoing")
	if err != nil {
		t.Fatalf("GetEdges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].EdgeType != "thematic" {
		t.Errorf("edge type = %q, want %q", edges[0].EdgeType, "thematic")
	}
	if edges[0].Weight != 0.8 {
		t.Errorf("weight = %f, want 0.8", edges[0].Weight)
	}
}

func TestDeleteNode(t *testing.T) {
	g := setupTestDB(t)

	id, _ := g.AddNode("concept", "temp", "private", nil)
	err := g.DeleteNode(id)
	if err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	node, _ := g.GetNode(id)
	if node != nil {
		t.Error("node should be deleted")
	}
}

func TestDeleteEdge(t *testing.T) {
	g := setupTestDB(t)

	id1, _ := g.AddNode("memory", "a", "private", nil)
	id2, _ := g.AddNode("memory", "b", "private", nil)
	edgeID, _ := g.AddEdge(id1, id2, "relates_to", nil, 1.0)

	err := g.DeleteEdge(edgeID)
	if err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}

	edges, _ := g.GetEdges(id1, "outgoing")
	if len(edges) != 0 {
		t.Error("edge should be deleted")
	}
}

func TestNeighbors(t *testing.T) {
	g := setupTestDB(t)

	id1, _ := g.AddNode("memory", "a", "private", nil)
	id2, _ := g.AddNode("memory", "b", "private", nil)
	id3, _ := g.AddNode("memory", "c", "private", nil)

	g.AddEdge(id1, id2, "thematic", nil, 1.0)
	g.AddEdge(id2, id3, "causal", nil, 0.5)

	// Direct neighbors
	neighbors, err := g.Neighbors(id1, "", 1)
	if err != nil {
		t.Fatalf("Neighbors: %v", err)
	}
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor, got %d", len(neighbors))
	}
	if neighbors[0].Node.ID != id2 {
		t.Errorf("neighbor id = %d, want %d", neighbors[0].Node.ID, id2)
	}

	// Depth 2 (bidirectional: b, c, and a as incoming neighbor of b)
	neighbors, _ = g.Neighbors(id1, "", 2)
	if len(neighbors) != 3 {
		t.Fatalf("expected 3 neighbors at depth 2, got %d", len(neighbors))
	}
}

func TestFindPath(t *testing.T) {
	g := setupTestDB(t)

	id1, _ := g.AddNode("memory", "start", "private", nil)
	id2, _ := g.AddNode("memory", "mid", "private", nil)
	id3, _ := g.AddNode("memory", "end", "private", nil)

	g.AddEdge(id1, id2, "thematic", nil, 1.0)
	g.AddEdge(id2, id3, "causal", nil, 1.0)

	path, err := g.FindPath(id1, id3, 5)
	if err != nil {
		t.Fatalf("FindPath: %v", err)
	}
	if !path.Found {
		t.Fatal("path should be found")
	}
	if len(path.Path) != 3 {
		t.Fatalf("expected 3 nodes in path, got %d", len(path.Path))
	}
	if path.Path[0].Label != "start" || path.Path[2].Label != "end" {
		t.Errorf("path labels wrong: %v", path.Path)
	}
}

func TestFindPathNoPath(t *testing.T) {
	g := setupTestDB(t)

	id1, _ := g.AddNode("memory", "a", "private", nil)
	id2, _ := g.AddNode("memory", "b", "private", nil)

	path, err := g.FindPath(id1, id2, 5)
	if err != nil {
		t.Fatalf("FindPath: %v", err)
	}
	if path.Found {
		t.Error("path should not be found")
	}
}

func TestSearchNodes(t *testing.T) {
	g := setupTestDB(t)

	g.AddNode("memory", "fixed nil pointer", "private", nil)
	g.AddNode("memory", "added error handling", "private", nil)
	g.AddNode("tool", "debugging-and-error-recovery", "shared", nil)

	// FTS search
	nodes, err := g.SearchNodes("nil pointer", "", "private", 10)
	if err != nil {
		t.Fatalf("SearchNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 result, got %d", len(nodes))
	}

	// Type filter
	nodes, _ = g.SearchNodes("", "tool", "shared", 10)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(nodes))
	}
}

func TestUpdateNodeProperties(t *testing.T) {
	g := setupTestDB(t)

	id, _ := g.AddNode("concept", "test", "private", map[string]string{"a": "1"})
	g.UpdateNodeProperties(id, map[string]string{"a": "2", "b": "3"})

	node, _ := g.GetNode(id)
	if node.Properties["a"] != "2" {
		t.Errorf("property a = %q, want %q", node.Properties["a"], "2")
	}
	if node.Properties["b"] != "3" {
		t.Errorf("property b = %q, want %q", node.Properties["b"], "3")
	}
}

func TestSharedScopeCrossDB(t *testing.T) {
	// Persona DB
	personaD, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open persona db: %v", err)
	}
	defer personaD.Close()

	// Shared DB
	sharedD, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open shared db: %v", err)
	}
	defer sharedD.Close()

	g := New(personaD.Conn())
	g.SetSharedDB(sharedD.Conn())

	// Add private node to persona DB
	privID, err := g.AddNode("concept", "private thing", "private", nil)
	if err != nil {
		t.Fatalf("AddNode private: %v", err)
	}

	// Add shared node to shared DB
	sharedID, err := g.AddNode("concept", "shared thing", "shared", nil)
	if err != nil {
		t.Fatalf("AddNode shared: %v", err)
	}

	// GetNodeInScope should find each in its own DB
	privNode, err := g.GetNodeInScope(privID, "private")
	if err != nil || privNode == nil {
		t.Fatalf("GetNodeInScope private: %v", err)
	}
	sharedNode, err := g.GetNodeInScope(sharedID, "shared")
	if err != nil || sharedNode == nil {
		t.Fatalf("GetNodeInScope shared: %v", err)
	}
	if sharedNode.Scope != "shared" {
		t.Errorf("scope = %q, want %q", sharedNode.Scope, "shared")
	}

	// SearchNodes with empty scope should find both
	nodes, err := g.SearchNodes("thing", "", "", 10)
	if err != nil {
		t.Fatalf("SearchNodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes from both DBs, got %d", len(nodes))
	}

	// SearchNodes scoped to private should only find private
	nodes, err = g.SearchNodes("thing", "", "private", 10)
	if err != nil {
		t.Fatalf("SearchNodes private: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 private node, got %d", len(nodes))
	}

	// SearchNodes scoped to shared should only find shared
	nodes, err = g.SearchNodes("thing", "", "shared", 10)
	if err != nil {
		t.Fatalf("SearchNodes shared: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 shared node, got %d", len(nodes))
	}

	// Delete shared node should work
	err = g.DeleteNodeInScope(sharedID, "shared")
	if err != nil {
		t.Fatalf("DeleteNodeInScope shared: %v", err)
	}
	node, _ := g.GetNodeInScope(sharedID, "shared")
	if node != nil {
		t.Error("shared node should be deleted")
	}

	// Private node should still exist
	node, _ = g.GetNodeInScope(privID, "private")
	if node == nil {
		t.Error("private node should still exist")
	}
}
