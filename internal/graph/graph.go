package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Node struct {
	ID         int64             `json:"id"`
	Type       string            `json:"type"`
	Label      string            `json:"label"`
	Properties map[string]string `json:"properties"`
	Scope      string            `json:"scope"`
	CreatedAt  string            `json:"created_at"`
	UpdatedAt  string            `json:"updated_at"`
}

type Edge struct {
	ID         int64             `json:"id"`
	SourceID   int64             `json:"source_id"`
	TargetID   int64             `json:"target_id"`
	EdgeType   string            `json:"edge_type"`
	Properties map[string]string `json:"properties"`
	Weight     float64           `json:"weight"`
	CreatedAt  string            `json:"created_at"`
}

type Graph struct {
	db       *sql.DB
	sharedDB *sql.DB
}

func New(db *sql.DB) *Graph {
	return &Graph{db: db}
}

func (g *Graph) SetSharedDB(shared *sql.DB) {
	g.sharedDB = shared
}

func (g *Graph) connForScope(scope string) *sql.DB {
	if scope == "shared" && g.sharedDB != nil {
		return g.sharedDB
	}
	return g.db
}

func (g *Graph) AddNode(nodeType, label, scope string, properties map[string]string) (int64, error) {
	if properties == nil {
		properties = make(map[string]string)
	}
	propJSON, err := json.Marshal(properties)
	if err != nil {
		return 0, fmt.Errorf("marshal properties: %w", err)
	}
	if scope == "" {
		scope = "private"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	conn := g.connForScope(scope)
	result, err := conn.Exec(
		"INSERT INTO graph_nodes (type, label, properties, scope, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		nodeType, label, string(propJSON), scope, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert node: %w", err)
	}
	return result.LastInsertId()
}

func (g *Graph) AddEdge(sourceID, targetID int64, edgeType string, properties map[string]string, weight float64) (int64, error) {
	if properties == nil {
		properties = make(map[string]string)
	}
	propJSON, err := json.Marshal(properties)
	if err != nil {
		return 0, fmt.Errorf("marshal properties: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := g.db.Exec(
		"INSERT INTO graph_edges (source_id, target_id, edge_type, properties, weight, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		sourceID, targetID, edgeType, string(propJSON), weight, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert edge: %w", err)
	}
	return result.LastInsertId()
}

func (g *Graph) scanNodeRow(conn *sql.DB, id int64) (*Node, error) {
	var n Node
	var propJSON string
	err := conn.QueryRow(
		"SELECT id, type, label, properties, scope, created_at, updated_at FROM graph_nodes WHERE id = ?", id,
	).Scan(&n.ID, &n.Type, &n.Label, &propJSON, &n.Scope, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(propJSON), &n.Properties); err != nil {
		n.Properties = make(map[string]string)
	}
	return &n, nil
}

func (g *Graph) GetNode(id int64) (*Node, error) {
	return g.GetNodeInScope(id, "")
}

func (g *Graph) GetNodeInScope(id int64, scope string) (*Node, error) {
	if scope == "shared" && g.sharedDB != nil {
		return g.scanNodeRow(g.sharedDB, id)
	}
	return g.scanNodeRow(g.db, id)
}

func (g *Graph) GetNodeByLabel(nodeType, label, scope string) (*Node, error) {
	var n Node
	var propJSON string
	conn := g.connForScope(scope)
	err := conn.QueryRow(
		"SELECT id, type, label, properties, scope, created_at, updated_at FROM graph_nodes WHERE type = ? AND label = ? AND scope = ?",
		nodeType, label, scope,
	).Scan(&n.ID, &n.Type, &n.Label, &propJSON, &n.Scope, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query node: %w", err)
	}
	if err := json.Unmarshal([]byte(propJSON), &n.Properties); err != nil {
		n.Properties = make(map[string]string)
	}
	return &n, nil
}

func (g *Graph) GetEdges(nodeID int64, direction string) ([]Edge, error) {
	var rows *sql.Rows
	var err error
	switch direction {
	case "outgoing":
		rows, err = g.db.Query(
			"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE source_id = ?", nodeID,
		)
	case "incoming":
		rows, err = g.db.Query(
			"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE target_id = ?", nodeID,
		)
	default:
		rows, err = g.db.Query(
			"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE source_id = ? OR target_id = ?", nodeID, nodeID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var e Edge
		var propJSON string
		if err := rows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.EdgeType, &propJSON, &e.Weight, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		if err := json.Unmarshal([]byte(propJSON), &e.Properties); err != nil {
			e.Properties = make(map[string]string)
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func (g *Graph) DeleteNode(id int64) error {
	_, err := g.db.Exec("DELETE FROM graph_nodes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	return nil
}

func (g *Graph) DeleteNodeInScope(id int64, scope string) error {
	conn := g.connForScope(scope)
	_, err := conn.Exec("DELETE FROM graph_nodes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	return nil
}

func (g *Graph) DeleteEdge(id int64) error {
	_, err := g.db.Exec("DELETE FROM graph_edges WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete edge: %w", err)
	}
	return nil
}

func (g *Graph) UpdateNodeProperties(id int64, properties map[string]string) error {
	propJSON, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := g.db.Exec("UPDATE graph_nodes SET properties = ?, updated_at = ? WHERE id = ?", string(propJSON), now, id)
	if err != nil {
		return fmt.Errorf("update node: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 && g.sharedDB != nil {
		_, err = g.sharedDB.Exec("UPDATE graph_nodes SET properties = ?, updated_at = ? WHERE id = ?", string(propJSON), now, id)
		if err != nil {
			return fmt.Errorf("update node (shared): %w", err)
		}
	}
	return nil
}
