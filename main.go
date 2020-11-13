package main

import "./vector"
import "fmt"

const VECSIZE = vector.VECSIZE

func main() {
	vec1 := [VECSIZE]int{1,1,0}
	vec2 := [VECSIZE]int{2,2,1}
	fmt.Println(vector.LTE(vec1, vec2))
}