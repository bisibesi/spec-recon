package common

import "spec-recon/internal/model"

// SortNodes separates a controller's children into business logic (mainStream) and utilities (utilStream).
// This function provides a unified, clean separation without adding any visual formatting.
//
// Logic:
//   - Traverse the tree using DFS
//   - Collect CTRL, SVC, MAP, SQL into mainStream
//   - Collect UTIL into utilStream
//   - Keep raw data clean (NO indentation characters like └, ㄴ)
//
// Returns:
//   - mainStream: Business logic nodes (Service, Mapper, SQL)
//   - utilStream: Utility nodes
func SortNodes(root *model.Node) (mainStream []*model.Node, utilStream []*model.Node) {
	var main []*model.Node
	var utils []*model.Node

	// Traverse children of the controller
	for _, child := range root.Children {
		traverseAndSort(child, &main, &utils)
	}

	return main, utils
}

// traverseAndSort recursively traverses the tree and separates nodes
func traverseAndSort(node *model.Node, main *[]*model.Node, utils *[]*model.Node) {
	if node == nil {
		return
	}

	// Classify node
	if node.Type == model.NodeTypeUtil {
		*utils = append(*utils, node)
		// For Utils, include their children in the util stream
		for _, child := range node.Children {
			traverseAndSort(child, utils, utils)
		}
	} else {
		// Business logic nodes (Service, Mapper, SQL)
		*main = append(*main, node)
		// Recursively process children
		for _, child := range node.Children {
			traverseAndSort(child, main, utils)
		}
	}
}
