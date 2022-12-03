# typed: true
# frozen_string_literal: true

class SignIn::Confirmations::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_confirmation = PhoneNumberConfirmation.find(session[:phone_number_confirmation_id])
    @form = SmsCodeForm.new(form_params.merge(phone_number_confirmation:))

    if @form.invalid?
      return render("sign_in/confirmations/new/call")
    end

    phone_number = PhoneNumber.find_by!(value: phone_number_confirmation.phone_number)

    ActiveRecord::Base.transaction do
      phone_number_confirmation.destroy
      sign_in(phone_number.user)
    end

    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:sms_code_form), ActionController::Parameters).permit(:sms_code)
  end
end
