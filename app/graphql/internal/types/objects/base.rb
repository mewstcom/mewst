# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::Base < GraphQL::Schema::Object
  edge_type_class(Internal::Types::Objects::Edges::Base)
  connection_type_class(Internal::Types::Objects::Connections::Base)
  field_class Internal::Types::Fields::Base
end
