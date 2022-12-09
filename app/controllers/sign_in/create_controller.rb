# typed: true
# frozen_string_literal: true

class SignIn::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = PhoneNumberForm.new(form_params)

    if @form.invalid?
      return render("sign_in/new/call")
    end

    result = CreatePhoneNumberConfirmationService.new(form: @form).call

    session[:phone_number_confirmation_id] = T.must(result.phone_number_confirmation).id
    flash[:success] = t("messages.authentication.confirmation_sms_sent")
    redirect_to sign_in_new_confirmation_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:phone_number_form), ActionController::Parameters).permit(:phone_number_origin, :phone_number)
  end
end
