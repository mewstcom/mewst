# typed: true
# frozen_string_literal: true

class Internal::Tasks::SendEmailConfirmationMailController < ApplicationController
  include Pubsub::Subscribable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    email_confirmation = EmailConfirmation.active.find(payload_params[:email_confirmation_id])

    EmailConfirmationMailer.email_confirmation(
      email_confirmation_id: email_confirmation.id,
      locale: payload_params[:locale]
    ).deliver_now

    head :no_content
  end

  private

  sig { returns(ActionController::Parameters) }
  def payload_params
    T.cast(params.require(:send_email_confirmation_mail), ActionController::Parameters).permit(:email_confirmation_id, :locale)
  end
end
