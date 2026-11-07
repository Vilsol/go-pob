package calculator2

// luaSort is LuaJIT's table.sort: Lua 5.1's quicksort, which is not stable, so elements the comparator ties come
// out in its order rather than the input's.
func luaSort[T any](a []T, less func(x, y T) bool) {
	luaAuxSort(a, 1, len(a), less)
}

func luaAuxSort[T any](a []T, l, u int, less func(x, y T) bool) {
	at := func(i int) T { return a[i-1] }
	swap := func(i, j int) { a[i-1], a[j-1] = a[j-1], a[i-1] }
	for l < u {
		if less(at(u), at(l)) {
			swap(l, u)
		}
		if u-l == 1 {
			break
		}
		i := (l + u) / 2
		if less(at(i), at(l)) {
			swap(i, l)
		} else if less(at(u), at(i)) {
			swap(i, u)
		}
		if u-l == 2 {
			break
		}
		p := at(i)
		swap(i, u-1)
		i, j := l, u-1
		for {
			for i++; less(at(i), p); i++ {
				if i > u {
					panic("invalid order function for sorting")
				}
			}
			for j--; less(p, at(j)); j-- {
				if j < l {
					panic("invalid order function for sorting")
				}
			}
			if j < i {
				break
			}
			swap(i, j)
		}
		swap(u-1, i)
		if i-l < u-i {
			j, i, l = l, i-1, i+1
		} else {
			j, i, u = i+1, u, i-1
		}
		luaAuxSort(a, j, i, less)
	}
}
