package ocrclient

func GroupTextBlocks(blocks []TextBlock, epsilonX, epsilonY int) [][]TextBlock {
	var groups [][]TextBlock
	used := make([]bool, len(blocks))

	for i := range blocks {
		if used[i] {
			continue
		}
		group := []TextBlock{blocks[i]}
		used[i] = true

		// Find neighbors of the current block
		for j := i + 1; j < len(blocks); j++ {
			if used[j] {
				continue
			}
			if isNeighbor(blocks[i].BoundingBox, blocks[j].BoundingBox, epsilonX, epsilonY) {
				group = append(group, blocks[j])
				used[j] = true
			}
		}

		// If the group only has one block, add it to a separate group
		if len(group) == 1 {
			groups = append(groups, []TextBlock{group[0]})
		} else {
			groups = append(groups, group)
		}
	}

	return groups
}

// isNeighbor returns true if two text blocks are neighbors, false otherwise
func isNeighbor(bb1, bb2 BoundingBox, epsilonX int, epsilonY int) bool {
	if abs(bb1.Left-bb2.Left) > epsilonX {
		return false
	}
	if abs(bb1.Top-bb2.Bottom) > epsilonY && abs(bb1.Bottom-bb2.Top) > epsilonY {
		return false
	}

	return true
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// max returns the maximum of a and b
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of a and b
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type TextBlockGroup []TextBlock

func (t TextBlockGroup) BoundingBox() BoundingBox {
	var minTop, maxBottom, minLeft, maxRight int
	for i, block := range t {
		if i == 0 {
			minTop = block.BoundingBox.Top
			maxBottom = block.BoundingBox.Bottom
			minLeft = block.BoundingBox.Left
			maxRight = block.BoundingBox.Right
		} else {
			if block.BoundingBox.Top < minTop {
				minTop = block.BoundingBox.Top
			}
			if block.BoundingBox.Bottom > maxBottom {
				maxBottom = block.BoundingBox.Bottom
			}
			if block.BoundingBox.Left < minLeft {
				minLeft = block.BoundingBox.Left
			}
			if block.BoundingBox.Right > maxRight {
				maxRight = block.BoundingBox.Right
			}
		}
	}
	return BoundingBox{
		Top:    minTop,
		Bottom: maxBottom,
		Left:   minLeft,
		Right:  maxRight,
	}
}
