package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type NeighborResult struct {
	Node      Node    `json:"node"`
	EdgeType  string  `json:"edge_type"`
	Direction string  `json:"direction"`
	Weight    float64 `json:"weight"`
}

type PathResult struct {
	Path  []Node `json:"path"`
	Edges []Edge `json:"edges"`
	Found bool   `json:"found"`
}

func (g *Graph) resolveNode(id int64) (*Node, error) {
	n, err := g.GetNode(id)
	if err != nil || n != nil {
		return n, err
	}
	if g.sharedDB != nil {
		return g.GetNodeInScope(id, "shared")
	}
	return nil, nil
}

func (g *Graph) queryNeighbors(conn *sql.DB, nodeID int64, edgeType string) ([]NeighborResult, []int64) {
	var results []NeighborResult
	var nextIDs []int64

	outRows, err := conn.Query(
		"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE source_id = ? AND (edge_type = ? OR ? = '')",
		nodeID, edgeType, edgeType,
	)
	if err != nil {
		return results, nextIDs
	}
	for outRows.Next() {
		var e Edge
		var propJSON string
		if err := outRows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.EdgeType, &propJSON, &e.Weight, &e.CreatedAt); err != nil {
			outRows.Close()
			return results, nextIDs
		}
		node, err := g.resolveNode(e.TargetID)
		if err != nil || node == nil {
			continue
		}
		results = append(results, NeighborResult{
			Node:      *node,
			EdgeType:  e.EdgeType,
			Direction: "outgoing",
			Weight:    e.Weight,
		})
		nextIDs = append(nextIDs, e.TargetID)
	}
	outRows.Close()

	inRows, err := conn.Query(
		"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE target_id = ? AND (edge_type = ? OR ? = '')",
		nodeID, edgeType, edgeType,
	)
	if err != nil {
		return results, nextIDs
	}
	for inRows.Next() {
		var e Edge
		var propJSON string
		if err := inRows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.EdgeType, &propJSON, &e.Weight, &e.CreatedAt); err != nil {
			inRows.Close()
			return results, nextIDs
		}
		node, err := g.resolveNode(e.SourceID)
		if err != nil || node == nil {
			continue
		}
		results = append(results, NeighborResult{
			Node:      *node,
			EdgeType:  e.EdgeType,
			Direction: "incoming",
			Weight:    e.Weight,
		})
		nextIDs = append(nextIDs, e.SourceID)
	}
	inRows.Close()
	return results, nextIDs
}

func (g *Graph) Neighbors(nodeID int64, edgeType string, maxDepth int) ([]NeighborResult, error) {
	if maxDepth <= 0 {
		maxDepth = 1
	}

	visited := make(map[int64]bool)
	var results []NeighborResult

	type queueItem struct {
		nodeID int64
		depth  int
	}
	queue := []queueItem{{nodeID, 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.nodeID] || item.depth >= maxDepth {
			continue
		}
		visited[item.nodeID] = true

		newResults, newQueue := g.queryNeighbors(g.db, item.nodeID, edgeType)
		results = append(results, newResults...)
		for _, nq := range newQueue {
			if !visited[nq] {
				queue = append(queue, queueItem{nq, item.depth + 1})
			}
		}
	}

	return results, nil
}

func (g *Graph) queryBFS(conn *sql.DB, sourceID, targetID int64, maxDepth int) (*PathResult, error) {
	type parent struct {
		nodeID int64
		edgeID int64
	}

	visited := make(map[int64]parent)
	queue := []int64{sourceID}
	visited[sourceID] = parent{-1, -1}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == targetID {
			var path []Node
			var edges []Edge
			cur := targetID
			for cur != sourceID {
				node, err := g.resolveNode(cur)
				if err != nil || node == nil {
					return &PathResult{Found: false}, nil
				}
				path = append([]Node{*node}, path...)
				p := visited[cur]
				if p.edgeID > 0 {
					edge, err := g.getEdge(p.edgeID)
					if err == nil && edge != nil {
						edges = append([]Edge{*edge}, edges...)
					}
				}
				cur = p.nodeID
			}
			srcNode, _ := g.resolveNode(sourceID)
			if srcNode != nil {
				path = append([]Node{*srcNode}, path...)
			}
			return &PathResult{Path: path, Edges: edges, Found: true}, nil
		}

		depth := 0
		for p := visited[current]; p.nodeID != -1; p = visited[p.nodeID] {
			depth++
		}
		if depth >= maxDepth {
			continue
		}

		outRows, err := conn.Query(
			"SELECT id, target_id FROM graph_edges WHERE source_id = ?", current,
		)
		if err != nil {
			continue
		}
		for outRows.Next() {
			var edgeID, target int64
			if err := outRows.Scan(&edgeID, &target); err != nil {
				outRows.Close()
				continue
			}
			if _, ok := visited[target]; !ok {
				visited[target] = parent{current, edgeID}
				queue = append(queue, target)
			}
		}
		outRows.Close()

		inRows, err := conn.Query(
			"SELECT id, source_id FROM graph_edges WHERE target_id = ?", current,
		)
		if err != nil {
			continue
		}
		for inRows.Next() {
			var edgeID, source int64
			if err := inRows.Scan(&edgeID, &source); err != nil {
				inRows.Close()
				continue
			}
			if _, ok := visited[source]; !ok {
				visited[source] = parent{current, edgeID}
				queue = append(queue, source)
			}
		}
		inRows.Close()
	}

	return &PathResult{Found: false}, nil
}

func (g *Graph) FindPath(sourceID, targetID int64, maxDepth int) (*PathResult, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}
	return g.queryBFS(g.db, sourceID, targetID, maxDepth)
}

func (g *Graph) getEdge(id int64) (*Edge, error) {
	var e Edge
	var propJSON string
	err := g.db.QueryRow(
		"SELECT id, source_id, target_id, edge_type, properties, weight, created_at FROM graph_edges WHERE id = ?", id,
	).Scan(&e.ID, &e.SourceID, &e.TargetID, &e.EdgeType, &propJSON, &e.Weight, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(propJSON), &e.Properties); err != nil {
		e.Properties = make(map[string]string)
	}
	return &e, nil
}

func (g *Graph) SearchNodes(query string, nodeType string, scope string, limit int) ([]Node, error) {
	if limit <= 0 {
		limit = 20
	}

	queryDB := func(conn *sql.DB, scopeFilter string) ([]Node, error) {
		var rows *sql.Rows
		var err error

		if query != "" {
			rows, err = conn.Query(
				`SELECT n.id, n.type, n.label, n.properties, n.scope, n.created_at, n.updated_at
				 FROM graph_nodes n
				 INNER JOIN graph_nodes_fts fts ON n.id = fts.rowid
				 WHERE graph_nodes_fts MATCH ?
				 AND (n.type = ? OR ? = '')
				 AND (n.scope = ? OR ? = '')
				 LIMIT ?`,
				query, nodeType, nodeType, scopeFilter, scopeFilter, limit,
			)
			if err == nil {
				var results []Node
				for rows.Next() {
					var n Node
					var propJSON string
					if scanErr := rows.Scan(&n.ID, &n.Type, &n.Label, &propJSON, &n.Scope, &n.CreatedAt, &n.UpdatedAt); scanErr != nil {
						rows.Close()
						return nil, scanErr
					}
					if jErr := json.Unmarshal([]byte(propJSON), &n.Properties); jErr != nil {
						n.Properties = make(map[string]string)
					}
					results = append(results, n)
				}
				rows.Close()
				if len(results) > 0 {
					return results, nil
				}
			}
			rows, err = conn.Query(
				`SELECT id, type, label, properties, scope, created_at, updated_at
				 FROM graph_nodes
				 WHERE (label LIKE ? OR properties LIKE ?)
				 AND (type = ? OR ? = '')
				 AND (scope = ? OR ? = '')
				 LIMIT ?`,
				"%"+query+"%", "%"+query+"%", nodeType, nodeType, scopeFilter, scopeFilter, limit,
			)
		} else {
			rows, err = conn.Query(
				`SELECT id, type, label, properties, scope, created_at, updated_at
				 FROM graph_nodes
				 WHERE (type = ? OR ? = '')
				 AND (scope = ? OR ? = '')
				 LIMIT ?`,
				nodeType, nodeType, scopeFilter, scopeFilter, limit,
			)
		}
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var nodes []Node
		for rows.Next() {
			var n Node
			var propJSON string
			if err := rows.Scan(&n.ID, &n.Type, &n.Label, &propJSON, &n.Scope, &n.CreatedAt, &n.UpdatedAt); err != nil {
				return nil, err
			}
			if err := json.Unmarshal([]byte(propJSON), &n.Properties); err != nil {
				n.Properties = make(map[string]string)
			}
			nodes = append(nodes, n)
		}
		return nodes, nil
	}

	if scope != "" {
		conn := g.connForScope(scope)
		return queryDB(conn, scope)
	}

	results, err := queryDB(g.db, "")
	if err != nil {
		return nil, fmt.Errorf("search nodes (persona): %w", err)
	}
	if g.sharedDB != nil {
		sharedResults, err := queryDB(g.sharedDB, "")
		if err == nil {
			results = append(results, sharedResults...)
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}
