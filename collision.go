package main

// Rect is a simple axis-aligned bounding box used for physics and hitboxes.
type Rect struct {
	X, Y, W, H float64
}

// Intersects reports whether two rectangles overlap.
func (r Rect) Intersects(o Rect) bool {
	return r.X < o.X+o.W && r.X+r.W > o.X && r.Y < o.Y+o.H && r.Y+r.H > o.Y
}

// Contains reports whether a point lies inside the rectangle.
func (r Rect) Contains(px, py float64) bool {
	return px >= r.X && px <= r.X+r.W && py >= r.Y && py <= r.Y+r.H
}

// Center returns the centre point of the rectangle.
func (r Rect) Center() (float64, float64) {
	return r.X + r.W/2, r.Y + r.H/2
}
