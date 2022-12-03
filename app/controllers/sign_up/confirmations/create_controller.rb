# typed: true
# frozen_string_literal: true

class SignUp::Confirmations::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_confirmation = PhoneNumberConfirmation.find(session[:phone_number_confirmation_id])
    @form = SmsCodeForm.new(form_params.merge(phone_number_confirmation:))

    if @form.invalid?
      return render("sign_up/confirmations/new/call")
    end

    result = CreatePhoneNumberService.new(form: @form).call

    session[:phone_number_id] = result.phone_number.id
    redirect_to new_user_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:sms_code_form), ActionController::Parameters).permit(:sms_code)
  end
end
