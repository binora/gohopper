package webapi

import "fmt"

type PointNotFoundError struct {
	Message string `json:"message"`
	Point   int    `json:"point_index"`
}

func (e PointNotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("Cannot find point %d", e.Point)
}

type PointOutOfBoundsError struct {
	Message string
	Point   int
}

func (e PointOutOfBoundsError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("Point %d is out of bounds", e.Point)
}

type ConnectionNotFoundError struct {
	Message string
}

func (e ConnectionNotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "Connection not found"
}

// ErrorBody follows the GraphHopper API message/hints style.
type ErrorBody struct {
	Message string      `json:"message"`
	Hints   []ErrorHint `json:"hints,omitempty"`
}

type ErrorHint struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func NewErrorBody(err error) ErrorBody {
	if err == nil {
		return ErrorBody{}
	}
	return ErrorBody{
		Message: err.Error(),
		Hints:   []ErrorHint{{Message: err.Error()}},
	}
}
