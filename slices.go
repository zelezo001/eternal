package eternal

func popFirst[S ~[]E, E any](slice S) (E, S) {
	return pop(slice, 0)
}

func popLast[S ~[]E, E any](slice S) (E, S) {
	return slice[len(slice)-1], slice[:len(slice)-1]
}

func pop[S ~[]E, E any](slice S, position uint) (E, S) {
	var popped = slice[position]
	for i := position; i < uint(len(slice)-1); i++ {
		slice[i] = slice[i+1]
	}

	return popped, slice[0 : len(slice)-1]
}

func swap[S ~[]E, E any](slice S, position uint, value E) E {
	old := slice[position]
	slice[position] = value
	return old
}

func prepend[S ~[]E, E any](slice S, value E) S {
	var emptyValue E
	slice = append(slice, emptyValue)
	for i := len(slice) - 1; i > 0; i-- {
		slice[i] = slice[i-1]
	}
	slice[0] = value

	return slice
}

// prependBefore
// Prepend value before target in array. Value must be present in slice, otherwise prependBefore panics.
// If target is present multiple times, value is prepended after the last one.
func prependBefore[S ~[]E, E comparable](slice S, value, target E) S {
	var emptyValue E
	var toReplace = len(slice)
	slice = append(slice, emptyValue)
	for toReplace > 0 {
		var copied = slice[toReplace-1]
		slice[toReplace] = copied
		if copied == target {
			toReplace--
			slice[toReplace] = value
			return slice
		}
		toReplace--
	}
	panic("target not found")
}
