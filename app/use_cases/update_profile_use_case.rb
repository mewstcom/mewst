# typed: strict
# frozen_string_literal: true

class UpdateProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :profile, Profile
  end

  sig do
    params(
      viewer: Actor,
      atname: String,
      name: String,
      description: String,
      avatar_kind: String,
      gravatar_email: String,
      image_url: String
    ).returns(Result)
  end
  def call(viewer:, atname:, name:, description:, avatar_kind:, gravatar_email:, image_url:)
    profile = viewer.profile.not_nil!
    gravatar_url = Gravatar.new(email: gravatar_email).url(size: 200)

    profile.update!(atname:, name:, description:, avatar_kind:, gravatar_email:, image_url:, gravatar_url:)

    Result.new(profile:)
  end
end
