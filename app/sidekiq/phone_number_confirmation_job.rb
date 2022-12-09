# typed: strict
# frozen_string_literal: true

class PhoneNumberConfirmationJob
  extend T::Sig

  include Sidekiq::Job

  sig { params(phone_number_confirmation_id: String).void }
  def perform(phone_number_confirmation_id)
    phone_number_confirmation = PhoneNumberConfirmation.find(phone_number_confirmation_id)

    client.messages.create(
      from: ENV.fetch("MEWST_TWILIO_PHONE_NUMBER"),
      to: phone_number_confirmation.phone_number,
      body: phone_number_confirmation.verification_code
    )
  rescue Twilio::REST::RestError => e
    Rails.logger.error(e)
  end

  private

  sig { returns(Twilio::REST::Client) }
  def client
    @client ||= T.let(Twilio::REST::Client.new(ENV.fetch("MEWST_TWILIO_ACCOUNT_SID"), ENV.fetch("MEWST_TWILIO_AUTH_TOKEN")), T.nilable(Twilio::REST::Client))
  end
end
