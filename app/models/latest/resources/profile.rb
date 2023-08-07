# typed: false
# frozen_string_literal: true

class Latest::Resources::Profile < Latest::Resources::Base
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :name
end
