# typed: strict
# frozen_string_literal: true

class UpdateUserUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, User
  end

  sig { params(user: User, locale: String, time_zone: String).returns(Result) }
  def call(user:, locale:, time_zone:)
    user.update!(locale:, time_zone:)

    Result.new(user:)
  end
end
