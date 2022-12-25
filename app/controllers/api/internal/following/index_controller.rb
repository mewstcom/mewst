# typed: true
# frozen_string_literal: true

class Api::Internal::Following::IndexController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    if params[:atnames].nil?
      render(json: {}, status: :not_found)
    end

    atnames = T.cast(params[:atnames], T::Array[String])
    follows = current_profile!.follows.where(target_profile: Profile.only_kept.where(atname: atnames))
    following_atnames = follows.map(&:target_profile).map { T.must(_1).atname }

    result = atnames.map do |atname|
      [atname, atname.in?(following_atnames)]
    end

    render(json: result.to_h)
  end
end
