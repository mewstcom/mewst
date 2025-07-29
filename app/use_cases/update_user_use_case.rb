# typed: strict
# frozen_string_literal: true

class UpdateUserUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, UserRecord
  end

  sig { params(viewer: ActorRecord, locale: String, time_zone: String).returns(Result) }
  def call(viewer:, locale:, time_zone:)
    viewer.user_record.not_nil!.update!(locale:, time_zone:)

    Result.new(user: viewer.user_record.not_nil!)
  end
end
