# typed: true
# frozen_string_literal: true

class PhoneNumbers::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_confirmation = PhoneNumberConfirmation.find(session[:phone_number_confirmation_id])
    @form = PhoneNumberForm.new(form_params.merge(phone_number_confirmation:))

    if @form.invalid?
      return render("phone_numbers/new/call")
    end

    result = CreatePhoneNumberService.new(form: @form).call

    session[:phone_number_id] = result.phone_number.id
    redirect_to new_user_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:phone_number_form), ActionController::Parameters).permit(:code)
  end
end
