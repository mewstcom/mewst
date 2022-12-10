# typed: strict
# frozen_string_literal: true

class SendPhoneNumberVerificationMessageJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(phone_number_verification_id: String).void }
  def perform(phone_number_verification_id)
    phone_number_verification = PhoneNumberVerification.find(phone_number_verification_id)

    body = "Confirmation code: #{phone_number_verification.confirmation_code}"
    if ENV["MEWST_TWILIO_SKIP"]
      puts body
    else
      client.messages.create(
        from: ENV.fetch("MEWST_TWILIO_PHONE_NUMBER"),
        to: phone_number_verification.phone_number,
        body:
      )
    end
  rescue Twilio::REST::RestError => e
    Rails.logger.error(e)
  end

  private

  sig { returns(Twilio::REST::Client) }
  def client
    @client ||= T.let(Twilio::REST::Client.new(ENV.fetch("MEWST_TWILIO_ACCOUNT_SID"), ENV.fetch("MEWST_TWILIO_AUTH_TOKEN")), T.nilable(Twilio::REST::Client))
  end
end
