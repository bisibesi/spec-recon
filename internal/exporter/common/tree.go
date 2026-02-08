package common

import "spec-recon/internal/model"

// FlattenedNode represents a node with its indentation level for display
type FlattenedNode struct {
	Node   *model.Node
	Indent int
}

// FlattenTree traverses the node tree and separates Util nodes from business logic nodes (Service, Mapper, SQL).
// It returns two slices: mainStream and utilStream.
func FlattenTree(root *model.Node) ([]*FlattenedNode, []*FlattenedNode) {
	var main []*FlattenedNode
	var utils []*FlattenedNode

	// Iterate children of the passed root
	for _, child := range root.Children {
		recursiveTraverse(child, 1, &main, &utils)
	}

	return main, utils
}

func recursiveTraverse(node *model.Node, indent int, main *[]*FlattenedNode, utils *[]*FlattenedNode) {
	row := &FlattenedNode{Node: node, Indent: indent}

	if node.Type == model.NodeTypeUtil {
		*utils = append(*utils, row)
		// For Utils, we include their children in the util stream to preserve context
		for _, child := range node.Children {
			recursiveTraverse(child, indent+1, utils, utils)
		}
	} else {
		*main = append(*main, row)
		for _, child := range node.Children {
			recursiveTraverse(child, indent+1, main, utils)
		}
	}
}
