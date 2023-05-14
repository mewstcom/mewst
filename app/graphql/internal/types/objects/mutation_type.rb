# typed: strict
# frozen_string_literal: true

class Internal::Types::Objects::MutationType < Internal::Types::Objects::Base
  # TODO: remove me
  field :test_field, String, null: false,
        description: "An example field added by the generator"
  def test_field
    "Hello World"
  end
end
