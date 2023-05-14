# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::Edges::Base < Internal::Types::Objects::Base
  # add `node` and `cursor` fields, as well as `node_type(...)` override
  include GraphQL::Types::Relay::EdgeBehaviors
end
