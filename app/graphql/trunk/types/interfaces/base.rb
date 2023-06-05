# typed: strict
# frozen_string_literal: true

module Trunk::Types::Interfaces::Base
  include GraphQL::Schema::Interface

  edge_type_class(Trunk::Types::BaseEdge)
  connection_type_class(Types::BaseConnection)

  field_class Types::BaseField
end
