# typed: strict
# frozen_string_literal: true

class Trunk::Mutations::UpdateProfile < Trunk::Mutations::Base
  argument :atname, String, required: false
  argument :avatar_url, String, required: false
  argument :description, String, required: false
  argument :name, String, required: false
  argument :profile_id, ID, required: true

  field :profile, Trunk::Types::Objects::ProfileType, null: true
  field :errors, [Trunk::Types::Objects::ClientErrorType], null: false

  def resolve(atname:, avatar_url:, description:, name:, profile_id:)
    form = Forms::Profile.new(atname:, avatar_url:, description:, name:)
    form.profile = Trunk::MewstSchema.object_from_id(profile_id)

    if form.invalid?
      return {
        profile: nil,
        errors: form.errors.full_messages.map { |message| {message:} }
      }
    end

    result = Commands::UpdateProfile.new(form:).call

    {
      profile: result.profile,
      errors: []
    }
  end
end
