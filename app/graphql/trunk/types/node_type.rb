# typed: strict
# frozen_string_literal: true

module Trunk::Types::NodeType
  include Trunk::Types::Interfaces::Base
  # Add the `id` field
  include GraphQL::Types::Relay::NodeBehaviors
end
