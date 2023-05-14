# typed: strict
# frozen_string_literal: true

class Internal::Types::Fields::Base < GraphQL::Schema::Field
  argument_class Internal::Types::Arguments::Base
end
