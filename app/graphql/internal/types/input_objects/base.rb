# typed: strict
# frozen_string_literal: true

class Internal::Types::InputObjects::Base < GraphQL::Schema::InputObject
  argument_class Internal::Types::Arguments::Base
end
