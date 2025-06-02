package model

type CommandSchema struct {
	Version  string    `json:"version"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Synopsis    string   `json:"synopsis"`
	Options     []Option `json:"options"`
}

type Option struct {
	Name        string   `json:"name"`
	Alias       []string `json:"alias"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Default     string   `json:"default"`
	Format      string   `json:"format"`
	Ref         string   `json:"$ref"`
}
