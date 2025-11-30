# typed: strict
# frozen_string_literal: true

class UpdatePasswordUseCase < ApplicationUseCase
  sig { params(email: String, password: String).void }
  def call(email:, password:)
    UserRecord.find_by!(email:).update!(password:)

    nil
  end
end
