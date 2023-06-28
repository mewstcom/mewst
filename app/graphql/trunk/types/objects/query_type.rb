# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::QueryType < Trunk::Types::Objects::Base
  field :profile, Trunk::Types::Objects::ProfileType, null: true do
    argument :atname, String, required: true
  end

  field :viewer, Trunk::Types::Objects::UserType,
        null: false,
        description: "The currently authenticated user."

  def profile(atname:)
    Profile.find_by(atname:)
  end

  def viewer
    context[:viewer]
  end
end
