# typed: strict
# frozen_string_literal: true

class Internal::Mutations::Base < GraphQL::Schema::RelayClassicMutation
  argument_class Internal::Types::Arguments::Base
  field_class Internal::Types::Fields::Base
  input_object_class Internal::Types::InputObjects::Base
  object_class Internal::Types::Objects::Base
end
