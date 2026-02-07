package llm

// OpenAICompatProvider wraps OpenAIProvider with provider metadata and a model
// list for validation. It exists to keep OpenAI-compatible providers (OpenRouter,
// Venice, Copilot, etc.) as thin constructors instead of duplicating boilerplate.
type OpenAICompatProvider struct {
	id     ProviderID
	name   string
	models []Model

	*OpenAIProvider
}

func newOpenAICompatProvider(apiKey, model, baseURL string, id ProviderID, name string, models []Model, defaultModel string) (*OpenAICompatProvider, error) {
	if model == "" {
		model = defaultModel
	}

	base, err := NewOpenAIProvider(apiKey, model, baseURL)
	if err != nil {
		return nil, err
	}

	return &OpenAICompatProvider{
		id:             id,
		name:           name,
		models:         models,
		OpenAIProvider: base,
	}, nil
}

func (p *OpenAICompatProvider) ID() ProviderID { return p.id }
func (p *OpenAICompatProvider) Name() string   { return p.name }
func (p *OpenAICompatProvider) Models() []Model {
	return p.models
}

func (p *OpenAICompatProvider) SetModel(modelID string) error {
	if err := ValidateModelID(modelID, p.models); err != nil {
		return err
	}
	p.model = modelID
	return nil
}
