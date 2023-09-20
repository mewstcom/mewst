# typed: strict
# frozen_string_literal: true

class UpdateProfileUseCase < ApplicationUseCase
  class Result < T::Struct
    const :profile, Profile
  end

  sig { params(profile: Profile, atname: String, avatar_url: String, description: String, name: String).returns(Result) }
  def call(profile:, atname:, avatar_url:, description:, name:)
    profile.update!(atname:, avatar_url:, description:, name:)

    Result.new(profile:)
  end
end
