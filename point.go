package main

type Point struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Metadata    Metadata `json:"metadata"`
	Value       Value    `json:"value"`
}

type Metadata struct {
	Unit       string  `json:"unit"`
	Divisor    float64 `json:"divisor"`
	Decimal    int     `json:"decimal"`
	IsWritable bool    `json:"isWritable"`
	VariableID int     `json:"variableId"`
}

type Value struct {
	IsOk         bool    `json:"isOk"`
	IntegerValue float64 `json:"integerValue"`
	StringValue  string  `json:"stringValue"`
}

// GetActualValue calculates the real number (e.g., 105 / 10 = 10.5)
func (p Point) GetActualValue() float64 {
	if p.Metadata.Divisor == 0 {
		return p.Value.IntegerValue
	}
	return p.Value.IntegerValue / p.Metadata.Divisor
}
