# typed: true
# frozen_string_literal: true

class Users::CreateController < ApplicationController
  include Authenticatable

  before_action :require_phone_number_verification_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
    @user_creator = @phone_number_verification.new_user_creator(
      idname: user_creator_params[:idname]
    )

    if @user_creator.invalid?
      return render("users/new/call")
    end

    @user_creator.call

    reset_session
    sign_in(T.must(@user_creator.user))

    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def user_creator_params
    T.cast(params.require(:user_creator), ActionController::Parameters).permit(:idname)
  end

  sig { returns(T.untyped) }
  def require_phone_number_verification_id
    unless session[:phone_number_verification_id]
      redirect_to root_path
    end
  end
end
