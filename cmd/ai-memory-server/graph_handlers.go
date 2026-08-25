package main

import (
	"fmt"
	"strconv"

	"github.com/coff33ninja/ai-memory/internal/graph"
)

func handleGraphAddNode(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	nodeType, _ := args["type"].(string)
	if nodeType == "" {
		return nil, fmt.Errorf("type is required")
	}
	label, _ := args["label"].(string)
	if label == "" {
		return nil, fmt.Errorf("label is required")
	}
	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "private"
	}

	var properties map[string]string
	if propRaw, ok := args["properties"].(map[string]interface{}); ok {
		properties = make(map[string]string)
		for k, v := range propRaw {
			properties[k] = fmt.Sprintf("%v", v)
		}
	}

	id, err := g.AddNode(nodeType, label, scope, properties)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Created node %d (type=%s, label=%s)", id, nodeType, label), nil
}

func handleGraphAddEdge(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	sourceID, err := toInt64(args["source_id"])
	if err != nil {
		return nil, fmt.Errorf("source_id is required")
	}
	targetID, err := toInt64(args["target_id"])
	if err != nil {
		return nil, fmt.Errorf("target_id is required")
	}
	edgeType, _ := args["edge_type"].(string)
	if edgeType == "" {
		return nil, fmt.Errorf("edge_type is required")
	}

	weight := 1.0
	if w, ok := args["weight"].(float64); ok {
		weight = w
	}

	var properties map[string]string
	if propRaw, ok := args["properties"].(map[string]interface{}); ok {
		properties = make(map[string]string)
		for k, v := range propRaw {
			properties[k] = fmt.Sprintf("%v", v)
		}
	}

	id, err := g.AddEdge(sourceID, targetID, edgeType, properties, weight)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Created edge %d (%d -> %d, type=%s)", id, sourceID, targetID, edgeType), nil
}

func handleGraphQuery(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	nodeType, _ := args["type"].(string)
	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "private"
	}
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	nodes, err := g.SearchNodes(query, nodeType, scope, limit)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return "No nodes found.", nil
	}

	var result string
	result = fmt.Sprintf("Found %d nodes:\n", len(nodes))
	for _, n := range nodes {
		result += fmt.Sprintf("- [%d] %s/%s (scope=%s)\n", n.ID, n.Type, n.Label, n.Scope)
	}
	return result, nil
}

func handleGraphNeighbors(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	nodeID, err := toInt64(args["node_id"])
	if err != nil {
		return nil, fmt.Errorf("node_id is required")
	}
	edgeType, _ := args["edge_type"].(string)
	maxDepth := 1
	if d, ok := args["max_depth"].(float64); ok {
		maxDepth = int(d)
	}

	results, err := g.Neighbors(nodeID, edgeType, maxDepth)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return "No neighbors found.", nil
	}

	var result string
	result = fmt.Sprintf("Found %d neighbors:\n", len(results))
	for _, r := range results {
		result += fmt.Sprintf("- [%d] %s/%s (via %s, %s, weight=%.2f)\n",
			r.Node.ID, r.Node.Type, r.Node.Label, r.EdgeType, r.Direction, r.Weight)
	}
	return result, nil
}

func handleGraphFindPath(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	sourceID, err := toInt64(args["source_id"])
	if err != nil {
		return nil, fmt.Errorf("source_id is required")
	}
	targetID, err := toInt64(args["target_id"])
	if err != nil {
		return nil, fmt.Errorf("target_id is required")
	}
	maxDepth := 5
	if d, ok := args["max_depth"].(float64); ok {
		maxDepth = int(d)
	}

	path, err := g.FindPath(sourceID, targetID, maxDepth)
	if err != nil {
		return nil, err
	}
	if !path.Found {
		return "No path found.", nil
	}

	var result string
	result = fmt.Sprintf("Path found (%d hops):\n", len(path.Path)-1)
	for i, n := range path.Path {
		result += fmt.Sprintf("  %d. [%d] %s/%s\n", i+1, n.ID, n.Type, n.Label)
	}
	if len(path.Edges) > 0 {
		result += "Edges:\n"
		for _, e := range path.Edges {
			result += fmt.Sprintf("  - %s (weight=%.2f)\n", e.EdgeType, e.Weight)
		}
	}
	return result, nil
}

func handleGraphDelete(g *graph.Graph, args map[string]interface{}) (interface{}, error) {
	nodeType, _ := args["type"].(string)
	scope, _ := args["scope"].(string)
	switch nodeType {
	case "node":
		id, err := toInt64(args["id"])
		if err != nil {
			return nil, fmt.Errorf("id is required for node deletion")
		}
		if scope != "" {
			if err := g.DeleteNodeInScope(id, scope); err != nil {
				return nil, err
			}
		} else {
			if err := g.DeleteNode(id); err != nil {
				return nil, err
			}
		}
		return fmt.Sprintf("Deleted node %d.", id), nil
	case "edge":
		id, err := toInt64(args["id"])
		if err != nil {
			return nil, fmt.Errorf("id is required for edge deletion")
		}
		if err := g.DeleteEdge(id); err != nil {
			return nil, err
		}
		return fmt.Sprintf("Deleted edge %d.", id), nil
	default:
		return nil, fmt.Errorf("type must be 'node' or 'edge'")
	}
}

func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(val), nil
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}
