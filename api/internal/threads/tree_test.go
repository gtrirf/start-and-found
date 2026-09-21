package threads

import (
	"testing"

	"github.com/google/uuid"

	"github.com/gtrirf/start-and-found/api/internal/posts"
)

func newPost(parent, root *uuid.UUID) posts.Post {
	return posts.Post{ID: uuid.New(), ParentID: parent, RootID: root}
}

func TestBuildAssemblesTree(t *testing.T) {
	root := newPost(nil, nil)
	firstReply := newPost(&root.ID, &root.ID)
	secondReply := newPost(&root.ID, &root.ID)
	nestedReply := newPost(&firstReply.ID, &root.ID)

	// The repository returns entries ordered by depth, then creation time.
	entries := []posts.ThreadEntry{
		{Post: firstReply, Depth: 1},
		{Post: secondReply, Depth: 1},
		{Post: nestedReply, Depth: 2},
	}

	tree := Build(root, entries)

	if tree.Post.ID != root.ID {
		t.Fatalf("root is %s, want %s", tree.Post.ID, root.ID)
	}
	if len(tree.Children) != 2 {
		t.Fatalf("root has %d children, want 2", len(tree.Children))
	}
	if tree.Children[0].Post.ID != firstReply.ID || tree.Children[0].Depth != 1 {
		t.Fatalf("first child is %s at depth %d", tree.Children[0].Post.ID, tree.Children[0].Depth)
	}
	if len(tree.Children[0].Children) != 1 {
		t.Fatalf("first reply has %d children, want 1", len(tree.Children[0].Children))
	}
	if tree.Children[0].Children[0].Depth != 2 {
		t.Fatalf("nested reply depth = %d, want 2", tree.Children[0].Children[0].Depth)
	}

	postCount, maxDepth := Stats(tree)
	if postCount != 4 {
		t.Fatalf("post count = %d, want 4", postCount)
	}
	if maxDepth != 2 {
		t.Fatalf("max depth = %d, want 2", maxDepth)
	}
}

func TestBuildKeepsRepliesOfMissingParents(t *testing.T) {
	root := newPost(nil, nil)
	missingParent := uuid.New()
	orphan := newPost(&missingParent, &root.ID)

	tree := Build(root, []posts.ThreadEntry{{Post: orphan, Depth: 1}})

	if len(tree.Children) != 1 {
		t.Fatalf("orphan reply was dropped: %d children", len(tree.Children))
	}
	if tree.Children[0].Post.ID != orphan.ID {
		t.Fatalf("unexpected child %s", tree.Children[0].Post.ID)
	}
	if tree.Children[0].Depth != 1 {
		t.Fatalf("orphan depth = %d, want 1", tree.Children[0].Depth)
	}
}

func TestStatsOfLeaf(t *testing.T) {
	root := newPost(nil, nil)
	postCount, maxDepth := Stats(Build(root, nil))
	if postCount != 1 || maxDepth != 0 {
		t.Fatalf("Stats(leaf) = (%d, %d), want (1, 0)", postCount, maxDepth)
	}
}

func TestStatsOfNil(t *testing.T) {
	if postCount, maxDepth := Stats(nil); postCount != 0 || maxDepth != 0 {
		t.Fatalf("Stats(nil) = (%d, %d), want (0, 0)", postCount, maxDepth)
	}
}
