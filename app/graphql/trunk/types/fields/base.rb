# typed: strict
# frozen_string_literal: true

class Trunk::Types::Fields::Base < GraphQL::Schema::Field
  argument_class Trunk::Types::Arguments::Base
end
