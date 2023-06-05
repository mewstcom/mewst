# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::QueryType < Trunk::Types::Objects::Base
  field :viewer, Trunk::Types::Objects::UserType,
        null: false,
        description: "The currently authenticated user."

  def viewer
    context[:viewer]
  end
end
