# typed: strict
# frozen_string_literal: true

class Trunk::Types::InputObjects::Base < GraphQL::Schema::InputObject
  argument_class Trunk::Types::Arguments::Base
end
