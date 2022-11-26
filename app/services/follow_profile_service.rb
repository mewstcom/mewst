# typed: strict
# frozen_string_literal: true

class FollowProfileService < ApplicationService
  class Error < BaseError; end
  class Result < BaseResult
    sig { returns(T.nilable(Profile)) }
    attr_reader :target_profile

    sig { params(errors: T::Array[Error], target_profile: T.nilable(Profile)).void }
    def initialize(errors: [], target_profile: nil)
      super(errors:)
      @target_profile = target_profile
    end
  end

  sig { returns(T.nilable(Profile)) }
  attr_accessor :source_profile

  sig { returns(T.nilable(Profile)) }
  attr_accessor :target_profile

  validates :source_profile, presence: true
  validates :target_profile, presence: true

  sig { returns(BaseResult) }
  def call
    if invalid?
      return validation_error_result(errors:)
    end

    follow = T.must(source_profile).follows.where(target_profile:).first_or_initialize

    if follow.persisted?
      return Result.new(target_profile:)
    end

    if follow.invalid?
      return validation_error_result(errors: follow.errors)
    end

    follow.save!

    Result.new(target_profile:)
  end
end
