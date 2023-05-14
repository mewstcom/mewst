# typed: strict
# frozen_string_literal: true

module Internal::Types::Interfaces::Base
  include GraphQL::Schema::Interface

  edge_type_class(Internal::Types::BaseEdge)
  connection_type_class(Types::BaseConnection)

  field_class Types::BaseField
end
