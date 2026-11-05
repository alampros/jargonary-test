export const editorYamlFixture = `# Owned by the platform team
version: "1" # Keep the format version quoted
name: 'Platform glossary'
entries:
  # Stable entry identity
  - id: api
    term: 'API' # Preferred name
    aliases:
      - text: Application programming interface
        sense: software
    senses:
      - id: software
        definition: |-
          A **stable** interface between services.

          Keep the contract clear.
        context: &shared_context Software development
        partOfSpeech: noun
        examples:
          - Call the API to retrieve an order.
        tags: [engineering, public]
      - id: product
        definition: A capability offered to developers.
        context: *shared_context
        partOfSpeech: noun
`
