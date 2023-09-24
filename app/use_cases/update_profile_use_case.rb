# typed: strict
# frozen_string_literal: true

class UpdateProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :profile, Profile
  end

  sig { params(viewer: Actor, atname: String, avatar_url: String, description: String, name: String).returns(Result) }
  def call(viewer:, atname:, avatar_url:, description:, name:)
    viewer.profile.not_nil!.update!(atname:, avatar_url:, description:, name:)

    Result.new(profile: viewer.profile.not_nil!)
  end
end
