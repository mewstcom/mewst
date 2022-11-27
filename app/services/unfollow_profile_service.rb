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
  end

  sig { returns(Result) }
  def call
    follow = source_profile.follows.find_by(target_profile:)

    follow&.destroy!

    Result.new(target_profile:)
  end

  sig { returns(Profile) }
  private def source_profile
    @source_profile ||= @form.source_profile
  end

  sig { returns(Profile) }
  private def target_profile
    @target_profile ||= @form.target_profile
  end
end
