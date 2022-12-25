# typed: true
# frozen_string_literal: true

class Api::Internal::Unfollow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(atname: params[:atname])

    unless current_profile!.following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    current_profile!.unfollow(target_profile:)

    render(json: {}, status: :ok)
  end
end
