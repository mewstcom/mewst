# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::Base < GraphQL::Schema::Object
  edge_type_class(Trunk::Types::Objects::Edges::Base)
  connection_type_class(Trunk::Types::Objects::Connections::Base)
  field_class Trunk::Types::Fields::Base
end
