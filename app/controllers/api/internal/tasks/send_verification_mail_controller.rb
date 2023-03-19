# typed: true
# frozen_string_literal: true

class Api::Internal::Tasks::SendVerificationMailController < ApplicationController
  include Pubsub::Subscribable

  skip_before_action :verify_authenticity_token
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    verification = Verification.active.find(payload_params[:verification_id])
    head 204
  end

  private

  sig { returns(ActionController::Parameters) }
  def payload_params
    T.cast(params.require(:send_verification_mail), ActionController::Parameters).permit(:verification_id)
  end
end
