# typed: strict
# frozen_string_literal: true

class UpdateUserUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, User
  end

  sig { params(user: User, locale: String).returns(Result) }
  def call(user:, locale:)
    user.update!(locale:)

    Result.new(user:)
  end
end
