# typed: strict
# frozen_string_literal: true

module Internal::Types::NodeType
  include Internal::Types::Interfaces::Base
  # Add the `id` field
  include GraphQL::Types::Relay::NodeBehaviors
end
