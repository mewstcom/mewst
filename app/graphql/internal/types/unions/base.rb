# typed: strict
# frozen_string_literal: true

class Internal::Types::Unions::Base < GraphQL::Schema::Union
  edge_type_class(Types::BaseEdge)
  connection_type_class(Types::BaseConnection)
end
