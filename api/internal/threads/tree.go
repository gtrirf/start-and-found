// Package threads renders the discussion tree of a post.
//
// The storage model is flat: every post carries parent_id and root_id. This
// package turns those rows back into the tree described in the README:
//
//	Post
//	├── Reply
//	│   ├── Reply
//	│   └── Reply
//	└── Reply
package threads

import (
	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/posts"
)

// Node is one post of a thread together with its replies.
type Node struct {
	Post     posts.Post `json:"post"`
	Depth    int        `json:"depth"`
	Children []*Node    `json:"children"`
}

// View is the payload of GET /v1/threads/{id}.
type View struct {
	Root      *Node `json:"root"`
	PostCount int   `json:"post_count"`
	MaxDepth  int   `json:"max_depth"`
}

// Build assembles the tree from a root post and its descendants.
//
// The input may arrive in any order; entries whose parent is missing from the
// slice (for example because the parent was deleted) are re-attached to the
// root so their content stays reachable.
func Build(root posts.Post, descendants []posts.ThreadEntry) *Node {
	rootNode := &Node{Post: root, Depth: 0, Children: []*Node{}}

	// Entries arrive ordered by increasing depth, so a parent node always exists
	// before its children are attached.
	nodes := map[uuid.UUID]*Node{root.ID: rootNode}
	for _, entry := range descendants {
		if entry.Post.ID == root.ID {
			continue
		}
		nodes[entry.Post.ID] = &Node{Post: entry.Post, Children: []*Node{}}
	}

	for _, entry := range descendants {
		if entry.Post.ID == root.ID {
			continue
		}
		node := nodes[entry.Post.ID]

		// A reply whose parent is missing (deleted) stays reachable by hanging off
		// the root instead of disappearing.
		parent := rootNode
		if entry.Post.ParentID != nil {
			if candidate, ok := nodes[*entry.Post.ParentID]; ok {
				parent = candidate
			}
		}
		node.Depth = parent.Depth + 1
		parent.Children = append(parent.Children, node)
	}

	return rootNode
}

// Stats counts the posts of a tree and returns its maximum depth.
func Stats(root *Node) (postCount int, maxDepth int) {
	if root == nil {
		return 0, 0
	}
	postCount = 1
	for _, child := range root.Children {
		childCount, childDepth := Stats(child)
		postCount += childCount
		if childDepth+1 > maxDepth {
			maxDepth = childDepth + 1
		}
	}
	return postCount, maxDepth
}
