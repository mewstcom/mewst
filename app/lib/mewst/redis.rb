# typed: strict
# frozen_string_literal: true

class Mewst::Redis
  extend T::Sig

  sig { returns(T.nilable(Redis)) }
  def self.client
    @redis ||= ConnectionPool::Wrapper.new do
      Redis.new(url: ENV.fetch("MEWST_REDIS_CACHE_URL"))
    end
  end
end
