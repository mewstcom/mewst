# typed: strict
# frozen_string_literal: true

class CreateSeedAccountUseCase < ApplicationUseCase
  sig { params(atname: String, email: String, locale: String, password: String, avatar_url: String).void }
  def call(atname:, email:, locale:, password:, avatar_url:)
    email_confirmation = EmailConfirmation.new(email:)
    account = Account.new(atname:, email:, locale:, password:)

    ActiveRecord::Base.transaction do
      email_confirmation.send_sign_up_email!(locale:)
      email_confirmation.success
      account.save!

      if avatar_url.present?
        account.profile.not_nil!.update!(avatar_url:)
      end
    end

    nil
  end
end
