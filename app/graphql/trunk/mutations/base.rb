# typed: strict
# frozen_string_literal: true

class Trunk::Mutations::Base < GraphQL::Schema::RelayClassicMutation
  argument_class Trunk::Types::Arguments::Base
  field_class Trunk::Types::Fields::Base
  input_object_class Trunk::Types::InputObjects::Base
  object_class Trunk::Types::Objects::Base
end
