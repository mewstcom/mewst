# typed: strict
# frozen_string_literal: true

class UnfollowProfileService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error]
    const :target_profile, T.nilable(Profile)
  end

  sig { returns(T.nilable(Profile)) }
  attr_accessor :source_profile

  sig { returns(T.nilable(Profile)) }
  attr_accessor :target_profile

  validates :source_profile, presence: true
  validates :target_profile, presence: true

  sig { returns(Result) }
  def call
    if invalid?
      return validation_error_result(errors:)
    end

    follow = T.must(source_profile).follows.find_by(target_profile:)

    unless follow
      return Result.new(errors: [], target_profile:)
    end

    follow.destroy!

    Result.new(errors: [], target_profile:)
  end

  sig { params(errors: ActiveModel::Errors).returns(Result) }
  private def validation_error_result(errors:)
    Result.new(errors: errors.map { |error| Error.new(message: error.full_message) })
  end
end
