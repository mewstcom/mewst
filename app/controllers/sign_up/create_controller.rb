# typed: true
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = PhoneNumberConfirmationForm.new(form_params)

    if @form.invalid?
      return render("sign_up/new/call")
    end

    result = ConfirmPhoneNumberService.new(form: @form).call

    session[:phone_number_confirmation_id] = result.phone_number_confirmation.id
    flash[:success] = t("messages.authentication.confirmation_sms_sent")
    redirect_to new_phone_number_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:phone_number_confirmation_form), ActionController::Parameters).permit(:phone_number, :phone_number_full)
  end
end
