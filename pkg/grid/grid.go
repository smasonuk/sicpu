package grid

func GetGridCoords(index int, cols int) (int, int) {
	return index % cols, index / cols
}
