# typed: strict
# frozen_string_literal: true

class UnfollowProfileService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
    const :target_profile, Profile
  end

  sig { params(form: FollowForm).void }
  def initialize(form:)
    @form = form
    @source_profile = T.let(T.must(@form.source_profile), Profile)
    @target_profile = T.let(T.must(@form.target_profile), Profile)
  end

  sig { returns(Result) }
  def call
    follow = source_profile.follows.find_by(target_profile:)

    follow&.destroy!

    Result.new(target_profile:)
  end

  private

  sig { returns(Profile) }
  attr_reader :source_profile

  sig { returns(Profile) }
  attr_reader :target_profile
end
