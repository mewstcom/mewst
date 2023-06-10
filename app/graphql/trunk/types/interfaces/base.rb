# typed: strict
# frozen_string_literal: true

module Trunk::Types::Interfaces::Base
  include GraphQL::Schema::Interface

  edge_type_class(Trunk::Types::Objects::Edges::Base)
  connection_type_class(Trunk::Types::Objects::Connections::Base)

  field_class Trunk::Types::Fields::Base
end
