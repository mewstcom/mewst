# typed: true
# frozen_string_literal: true

class Api::Internal::Follow::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    target_profile = Profile.only_kept.find_by!(atname: params[:atname])

    if current_profile!.following?(target_profile:)
      return render(json: {}, status: :ok)
    end

    current_profile!.follow(target_profile:)

    render(json: {}, status: :created)
  end
end
