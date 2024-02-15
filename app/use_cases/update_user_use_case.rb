# typed: strict
# frozen_string_literal: true

class UpdateUserUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, User
  end

  sig { params(viewer: Actor, locale: String, time_zone: String).returns(Result) }
  def call(viewer:, locale:, time_zone:)
    viewer.user.update!(locale:, time_zone:)

    Result.new(user: viewer.user)
  end
end
